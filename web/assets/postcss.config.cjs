const cssnano = require("cssnano")

module.exports = {
  plugins: [
	require("postcss-import"),
    require("postcss-url")({
        url: "copy",
        useHash: true,
        assetsPath: "misc",
		hashOptions: {append: true}
    }),
      require("autoprefixer"),
	process.env.NODE_ENV === 'production' ? cssnano({preset: "default"}) : null,
  ]
}

