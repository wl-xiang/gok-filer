/**
 * Gokapi theme switcher.
 *
 * The selected theme is stored in localStorage and applied to the
 * data-bs-theme attribute of the <html> element, which both Bootstrap 5.3 and
 * ./css/theme.css use to resolve their colour variables.
 *
 * This file is loaded synchronously in <head> so the stored theme is applied
 * before the first paint (no flash of the wrong theme).
 */
(function () {
	"use strict";

	var STORAGE_KEY = "gokapi-theme";
	var LIGHT = "light";
	var DARK = "dark";
	var root = document.documentElement;

	function isTheme(value) {
		return value === LIGHT || value === DARK;
	}

	function storedTheme() {
		try {
			var value = window.localStorage.getItem(STORAGE_KEY);
			return isTheme(value) ? value : null;
		} catch (err) {
			return null;
		}
	}

	function currentTheme() {
		var attribute = root.getAttribute("data-bs-theme");
		return attribute === LIGHT ? LIGHT : DARK;
	}

	function updateToggleIcon(theme) {
		var icon = document.getElementById("gk-theme-icon");
		if (!icon) {
			return;
		}
		icon.className = theme === LIGHT ? "bi bi-sun-fill" : "bi bi-moon-stars-fill";
	}

	function applyTheme(theme) {
		root.setAttribute("data-bs-theme", theme);
		updateToggleIcon(theme);
	}

	// Apply the saved theme as early as possible
	var initial = storedTheme();
	applyTheme(initial === null ? currentTheme() : initial);

	// Expose the toggle used by the toolbar button
	window.gokapiToggleTheme = function () {
		var next = currentTheme() === LIGHT ? DARK : LIGHT;
		try {
			window.localStorage.setItem(STORAGE_KEY, next);
		} catch (err) {
			/* storage may be unavailable, the theme is still applied for this page */
		}
		applyTheme(next);
	};

	// The toggle button only exists once the body has been parsed
	document.addEventListener("DOMContentLoaded", function () {
		updateToggleIcon(currentTheme());
	});

	// Close the user dropdown when clicking anywhere else. Needed as a
	// fallback because disabled form controls swallow click events entirely
	// (the event never reaches document, so Bootstrap's own outside-click
	// close never fires). mousedown in the capture phase always arrives.
	document.addEventListener("mousedown", function (event) {
		if (!window.bootstrap) {
			return;
		}
		var menu = document.querySelector(".gk-user-dropdown .dropdown-menu.show");
		if (!menu || menu.contains(event.target)) {
			return;
		}
		var toggle = document.querySelector(".gk-user-menu");
		if (toggle && !toggle.contains(event.target)) {
			window.bootstrap.Dropdown.getOrCreateInstance(toggle).hide();
		}
	}, true);

	// Escape closes the user dropdown as well, regardless of focus
	document.addEventListener("keydown", function (event) {
		if (event.key !== "Escape" || !window.bootstrap) {
			return;
		}
		var toggle = document.querySelector(".gk-user-menu");
		if (toggle && document.querySelector(".gk-user-dropdown .dropdown-menu.show")) {
			window.bootstrap.Dropdown.getOrCreateInstance(toggle).hide();
		}
	}, true);

	// Dropdown menus inside horizontally scrollable tables (.table-responsive)
	// get clipped: overflow-x: auto also clips vertically, so a menu opening
	// below the last row is cut off. Fix: reposition the OPEN menu with
	// position: fixed, which escapes the scroll container's clipping. The
	// glass card's backdrop-filter provides the containing block, so offsets
	// are computed relative to the card, not the viewport.
	// Implemented as a delegated click listener (click events always bubble,
	// unlike Bootstrap's custom dropdown events).
	//
	// The action buttons sit in the STICKY actions column (z-index: 2). Rows
	// below paint their own sticky cells at the same level but later in the
	// DOM, so they would cover the menu of any row above the last one. While
	// a menu is open, its cell is lifted above all other cells.
	function gkClearOpenMenuCells() {
		var marked = document.querySelectorAll(".gk-menu-open-cell");
		for (var i = 0; i < marked.length; i++) {
			marked[i].classList.remove("gk-menu-open-cell");
		}
	}

	document.addEventListener("click", function (event) {
		gkClearOpenMenuCells();
		var toggle = event.target.closest ? event.target.closest("[data-bs-toggle='dropdown']") : null;
		if (!toggle || !toggle.closest(".table-responsive")) {
			return;
		}
		// Bootstrap opens the menu synchronously during this click; reposition
		// on the next tick once it and Popper have finished
		window.setTimeout(function () {
			var menu = toggle.parentElement.querySelector(".dropdown-menu.show");
			if (!menu) {
				return;
			}
			var cell = menu.closest("td") || menu.closest("th");
			if (cell) {
				cell.classList.add("gk-menu-open-cell");
			}
			var card = toggle.closest(".card");
			var base = card
				? card.getBoundingClientRect()
				: { left: 0, top: 0, right: window.innerWidth };
			var tRect = toggle.getBoundingClientRect();
			menu.style.position = "fixed";
			menu.style.transform = "none";
			menu.style.margin = "0";
			if (menu.classList.contains("dropdown-menu-end")) {
				menu.style.left = "auto";
				menu.style.right = Math.max(0, base.right - tRect.right) + "px";
			} else {
				menu.style.right = "auto";
				menu.style.left = (tRect.left - base.left) + "px";
			}
			menu.style.top = (tRect.bottom - base.top + 2) + "px";
		}, 0);
	}, true);

	// A scrolled table would drift away from the fixed menu, so close any
	// open table dropdown as soon as something is scrolled
	document.addEventListener("scroll", function () {
		if (!window.bootstrap) {
			return;
		}
		gkClearOpenMenuCells();
		var open = document.querySelectorAll(".table-responsive .dropdown-menu.show");
		for (var i = 0; i < open.length; i++) {
			var group = open[i].parentElement;
			var toggle = group ? group.querySelector("[data-bs-toggle='dropdown']") : null;
			var inst = toggle ? window.bootstrap.Dropdown.getInstance(toggle) : null;
			if (inst) {
				inst.hide();
			}
		}
	}, true);
})();
