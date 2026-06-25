/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,js,ts}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Satoshi", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "monospace"],
      },
      colors: {
        bg: "#000808",
        card: {
          from: "#002F2F",
          to: "#001F1F",
          light: "#003D3D",
        },
        accent: {
          DEFAULT: "#2CD1D1",
          hover: "#36E2E2",
          pressed: "#24ABAB",
        },
        eth: {
          DEFAULT: "#627EEA",
          muted: "#8A92B2",
        },
      },
      keyframes: {
        "fade-in": {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        glow: {
          "0%, 100%": { boxShadow: "0 0 20px rgba(44, 209, 209, 0.3)" },
          "50%": { boxShadow: "0 0 30px rgba(44, 209, 209, 0.6)" },
        },
        "pulse-dot": {
          "0%, 100%": { opacity: "1", boxShadow: "0 0 0 0 rgba(44, 209, 209, 0.5)" },
          "50%": { opacity: "0.7", boxShadow: "0 0 0 5px rgba(44, 209, 209, 0)" },
        },
      },
      animation: {
        "fade-in": "fade-in 0.3s ease-out",
        glow: "glow 3s ease-in-out infinite",
        "pulse-dot": "pulse-dot 2s ease-in-out infinite",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};
