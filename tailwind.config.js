/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["../../**/*.{go,js,templ,html,css}"],
    theme: {
        container: {
            center: true,
            padding: {
                DEFAULT: "1rem",
                mobile: "2rem",
                tablet: "4rem",
                desktop: "5rem",
            },
        },
        extend: {
            colors: {
                extend: {},
            },
            borderRadius: {
                lg: "var(--radius)",
                md: "calc(var(--radius) - 2px)",
                sm: "calc(var(--radius) - 4px)",
            },
        },
    },
    plugins: [
        require("@tailwindcss/forms"),
        require("@tailwindcss/typography"),
    ],
};
