// sbclaim.c — App-Sandbox USB claim test for gemina task #2564.
//
// Usage: sbclaim            (run the same binary signed with / without the
//                            App Sandbox entitlement and compare output)
// Steps, each reporting the raw return code:
//   0. Report whether this process is sandboxed (APP_SANDBOX_CONTAINER_ID,
//      and whether ~/Desktop is listable — denied inside the sandbox).
//   1. IOKit: find the RNDIS control interface (IOUSBHostInterface, class
//      224/1/3) and USBInterfaceOpen it via IOUSBInterfaceInterface → IOReturn.
//   2. libusb (the spike's path): open device by class, claim ctrl+data,
//      REMOTE_NDIS_INITIALIZE → libusb codes. Also tries NCM (class 2/13).
// Output is redacted: no serial, MAC, IP, VID/PID printed.
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOCFPlugIn.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/usb/IOUSBLib.h>
#include <dirent.h>
#include <errno.h>
#include <libusb-1.0/libusb.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static int cfnum(io_service_t s, const char *key) {
    CFStringRef k = CFStringCreateWithCString(NULL, key, kCFStringEncodingUTF8);
    CFTypeRef v = IORegistryEntryCreateCFProperty(s, k, NULL, 0);
    CFRelease(k);
    int out = -1;
    if (v && CFGetTypeID(v) == CFNumberGetTypeID())
        CFNumberGetValue(v, kCFNumberIntType, &out);
    if (v) CFRelease(v);
    return out;
}

// Step 1: raw IOKit open of the first interface matching cls/sub.
static void iokit_open(int cls, int sub, const char *label) {
    io_iterator_t it;
    kern_return_t kr = IOServiceGetMatchingServices(
        kIOMainPortDefault, IOServiceMatching("IOUSBHostInterface"), &it);
    if (kr != KERN_SUCCESS) { printf("IOKIT %s match kr=0x%08x\n", label, kr); return; }
    io_service_t s; int found = 0;
    while ((s = IOIteratorNext(it))) {
        if (cfnum(s, "bInterfaceClass") == cls && cfnum(s, "bInterfaceSubClass") == sub) {
            found = 1;
            IOCFPlugInInterface **plug = NULL; SInt32 score;
            kr = IOCreatePlugInInterfaceForService(s, kIOUSBInterfaceUserClientTypeID,
                                                   kIOCFPlugInInterfaceID, &plug, &score);
            printf("IOKIT %s plugin kr=0x%08x\n", label, kr);
            if (kr == KERN_SUCCESS && plug) {
                IOUSBInterfaceInterface300 **intf = NULL;
                HRESULT hr = (*plug)->QueryInterface(plug,
                    CFUUIDGetUUIDBytes(kIOUSBInterfaceInterfaceID300), (LPVOID *)&intf);
                (*plug)->Release(plug);
                printf("IOKIT %s queryinterface hr=0x%08x\n", label, (unsigned)hr);
                if (hr == 0 && intf) {
                    IOReturn r = (*intf)->USBInterfaceOpen(intf);
                    printf("IOKIT %s USBInterfaceOpen ior=0x%08x (%s)\n", label, r,
                           r == kIOReturnSuccess ? "kIOReturnSuccess" :
                           r == kIOReturnNotPermitted ? "kIOReturnNotPermitted" :
                           r == kIOReturnExclusiveAccess ? "kIOReturnExclusiveAccess" :
                           r == kIOReturnNotPrivileged ? "kIOReturnNotPrivileged" : "other");
                    if (r == kIOReturnSuccess) (*intf)->USBInterfaceClose(intf);
                    (*intf)->Release(intf);
                }
            }
            IOObjectRelease(s);
            continue;
        }
        IOObjectRelease(s);
    }
    IOObjectRelease(it);
    if (!found) printf("IOKIT %s interface not present\n", label);
}

// Step 2: libusb open + claim + (RNDIS) INITIALIZE, keyed on class.
static void libusb_path(int want) {
    libusb_context *ctx = NULL;
    int rc = libusb_init(&ctx);
    printf("LIBUSB init rc=%d\n", rc);
    if (rc) return;
    libusb_device **list; ssize_t n = libusb_get_device_list(ctx, &list);
    printf("LIBUSB device_list n=%zd\n", n);
    int done = 0;
    for (ssize_t i = 0; i < n; i++) {
        struct libusb_config_descriptor *cfg;
        if (libusb_get_active_config_descriptor(list[i], &cfg) != 0) continue;
        int c = -1, d = -1, kind = 0; // kind 1=rndis 2=ncm
        for (int j = 0; j < cfg->bNumInterfaces; j++) {
            const struct libusb_interface_descriptor *id = &cfg->interface[j].altsetting[0];
            if (id->bInterfaceClass == 0xE0 && id->bInterfaceSubClass == 1) { c = id->bInterfaceNumber; kind = 1; }
            if (id->bInterfaceClass == 0x02 && id->bInterfaceSubClass == 0x0D) { c = id->bInterfaceNumber; kind = 2; }
            if (id->bInterfaceClass == 0x0A && d < 0) d = id->bInterfaceNumber;
        }
        libusb_free_config_descriptor(cfg);
        if (c < 0 || kind != want) continue;
        done++;
        const char *k = kind == 1 ? "rndis" : "ncm";
        libusb_device_handle *h = NULL;
        rc = libusb_open(list[i], &h);
        printf("LIBUSB %s open rc=%d (%s)\n", k, rc, libusb_error_name(rc));
        if (rc) continue;
        int rcc = libusb_claim_interface(h, c);
        int rcd = d >= 0 ? libusb_claim_interface(h, d) : -99;
        printf("LIBUSB %s claim ctrl rc=%d (%s) data rc=%d (%s)\n", k, rcc,
               libusb_error_name(rcc), rcd, rcd == -99 ? "n/a" : libusb_error_name(rcd));
        if (kind == 1 && rcc == 0) {
            uint8_t init[24] = {0}, resp[1025] = {0};
            uint32_t f[6] = {2, 24, 1, 1, 0, 0x4000};
            for (int q = 0; q < 6; q++) for (int b = 0; b < 4; b++) init[q*4+b] = (f[q] >> (8*b)) & 0xff;
            int s = libusb_control_transfer(h, 0x21, 0, 0, c, init, 24, 1000);
            usleep(50000);
            int g = libusb_control_transfer(h, 0xA1, 1, 0, c, resp, 1024, 1000);
            uint32_t mt = resp[0] | resp[1]<<8 | resp[2]<<16 | (uint32_t)resp[3]<<24;
            uint32_t st = resp[12] | resp[13]<<8 | resp[14]<<16 | (uint32_t)resp[15]<<24;
            printf("LIBUSB rndis INITIALIZE send=%d recv=%d type=0x%08x status=0x%08x -> %s\n",
                   s, g, mt, st, (g >= 16 && mt == 0x80000002u && st == 0) ? "INIT_CMPLT OK" : "FAIL");
            // SET OID_GEN_CURRENT_PACKET_FILTER (DIRECTED|MULTICAST|BROADCAST), then one
            // bulk-IN read: proves the data endpoints are usable (timeout = no traffic, not denial).
            uint8_t set[32] = {0};
            uint32_t sf[8] = {5, 32, 2, 0x0001010E, 4, 20, 0, 0x13};
            for (int q = 0; q < 8; q++) for (int b = 0; b < 4; b++) set[q*4+b] = (sf[q] >> (8*b)) & 0xff;
            memset(resp, 0, sizeof resp);
            s = libusb_control_transfer(h, 0x21, 0, 0, c, set, 32, 1000);
            usleep(20000);
            g = libusb_control_transfer(h, 0xA1, 1, 0, c, resp, 1024, 1000);
            mt = resp[0] | resp[1]<<8 | resp[2]<<16 | (uint32_t)resp[3]<<24;
            st = resp[12] | resp[13]<<8 | resp[14]<<16 | (uint32_t)resp[15]<<24;
            printf("LIBUSB rndis SET_PACKET_FILTER send=%d recv=%d type=0x%08x status=0x%08x -> %s\n",
                   s, g, mt, st, (g >= 16 && mt == 0x80000005u && st == 0) ? "SET_CMPLT OK" : "FAIL");
            uint8_t ep_in = 0; struct libusb_config_descriptor *cf2;
            if (libusb_get_active_config_descriptor(list[i], &cf2) == 0) {
                for (int j = 0; j < cf2->bNumInterfaces; j++) {
                    const struct libusb_interface_descriptor *id = &cf2->interface[j].altsetting[0];
                    if (id->bInterfaceNumber == d)
                        for (int e2 = 0; e2 < id->bNumEndpoints; e2++)
                            if (id->endpoint[e2].bEndpointAddress & 0x80) ep_in = id->endpoint[e2].bEndpointAddress;
                }
                libusb_free_config_descriptor(cf2);
            }
            uint8_t bb[2048]; int got = 0;
            int br = libusb_bulk_transfer(h, ep_in, bb, sizeof bb, &got, 800);
            printf("LIBUSB rndis bulk-IN rc=%d (%s) bytes=%d\n", br, libusb_error_name(br), got);
        }
        if (rcc == 0) libusb_release_interface(h, c);
        if (rcd == 0) libusb_release_interface(h, d);
        libusb_close(h);
    }
    if (!done) printf("LIBUSB no %s function found\n", want == 1 ? "rndis" : "ncm");
    libusb_free_device_list(list, 1);
    libusb_exit(ctx);
}

int main(void) {
    const char *cid = getenv("APP_SANDBOX_CONTAINER_ID");
    char p[1024]; snprintf(p, sizeof p, "%s/Desktop", getenv("HOME") ? getenv("HOME") : "");
    DIR *dd = opendir(p); int e = errno;
    printf("SANDBOX container_id=%s desktop_listable=%s%s\n", cid ? "set" : "unset",
           dd ? "yes" : "no", dd ? "" : (e == EPERM ? " (EPERM)" : ""));
    if (dd) closedir(dd);
    iokit_open(224, 1, "rndis-ctrl");
    iokit_open(2, 13, "ncm-ctrl");
    libusb_path(1);
    libusb_path(2);
    return 0;
}
