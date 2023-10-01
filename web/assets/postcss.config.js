const cssnano = require("cssnano")

module.exports = {
  plugins: [
	require("postcss-import"),
	require("tailwindcss/nesting"),
    require("postcss-url")({
        url: "copy",
        useHash: true,
        assetsPath: "misc"
    }),
    require("tailwindcss"),
	require("autoprefixer"),
	process.env.NODE_ENV === 'production' ? cssnano({preset: "default"}) : null,
  ]
}

