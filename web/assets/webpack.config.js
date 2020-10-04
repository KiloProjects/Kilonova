const cssextractor = require('mini-css-extract-plugin')
const purgecss = require('purgecss-webpack-plugin')
const copy = require('copy-webpack-plugin')
const glob = require('glob')

module.exports = {
	entry: {
		app: './src/index.js',
		styles: './src/styles.scss',
	},
	output: {
		path: __dirname + '/../../static'
	},
	plugins: [
		new cssextractor(),
		//new purgecss({paths: glob.sync(`../templ/**/*`, {nodir:true})}),
		new copy({
			patterns: [
				{ from: './public/favicon.ico' }
			]
		})
	],
	module: {
		rules: [
			{
				test: /\.css$/i,
				use: [
					cssextractor.loader,
					//'style-loader',
					'css-loader',
					'postcss-loader'
				]
			},
			{
				test: /\.sass$|\.scss$/i,
				use: [
					cssextractor.loader,
					{
						loader: 'css-loader',
						options: {
							importLoaders: 1
						}
					},
					{
						loader: 'postcss-loader',
						options: {
							sourceMap: true,
						}
					},
					{
						loader: 'sass-loader?sourceMap'
					}
				]
			},
			{
				test: /\.(woff(2)?|ttf|eot|svg)(\?v=\d+\.\d+\.\d+)?$/,
				use: [
					{
						loader: 'file-loader',
						options: {
							name: '[name].[ext]',
							outputPath: 'fonts/'
						}
					}
				]
			},
		]
	}
};
