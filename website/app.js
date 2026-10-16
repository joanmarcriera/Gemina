/* Gemina VPN — tiny progressive-enhancement script.
   1. Cycles the hero "scope" so visitors can watch one input link
      drop while the OUTPUT trace stays unbroken — the core promise.
   Everything degrades gracefully: with JS off, the page is complete
   and the scope simply shows both links up. */
(function () {
  "use strict";

  var reduceMotion = window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---- 1. Hero scope: demonstrate seamless failover ---- */
  var wifi = document.querySelector('[data-trace="wifi"]');
  var phone = document.querySelector('[data-trace="phone"]');
  var status = document.querySelector('[data-live]');

  if (wifi && phone && status && !reduceMotion) {
    // states: which input is "down". OUTPUT is never touched.
    var states = [
      { down: null,    label: "both paths up",        state: "" },
      { down: phone,   label: "phone dropped — holding on Wi-Fi", state: "phone-down" },
      { down: null,    label: "both paths up",        state: "" },
      { down: wifi,    label: "Wi-Fi dropped — holding on phone", state: "wifi-down" }
    ];
    var i = 0;
    setInterval(function () {
      i = (i + 1) % states.length;
      var s = states[i];
      wifi.classList.toggle("is-down", s.down === wifi);
      phone.classList.toggle("is-down", s.down === phone);
      status.textContent = s.label;
      if (s.state) {
        status.setAttribute("data-state", s.state);
      } else {
        status.removeAttribute("data-state");
      }
    }, 2600);
  }
})();
