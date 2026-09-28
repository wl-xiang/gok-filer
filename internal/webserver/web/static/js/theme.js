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
})();
