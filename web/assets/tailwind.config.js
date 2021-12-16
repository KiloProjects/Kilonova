const colors = require('tailwindcss/colors')
//const glob = require('fast-glob');
//glob('../**/*.{html,go,js}').then(val => console.log(val));

module.exports = {
  	content: [
		'../templ/**/*.html',
		'../templ/*.html',

		'../*.go',
	  	'./*.{js,ts,jsx,tsx}',
		'./components/*.{js,ts,jsx,tsx}',
  	],
  	theme: {
    	extend: {
			colors: {
				gray: colors.zinc,
			}
		},
		fontSize: {
			xs: '0.75rem',
      		sm: '0.875rem',
      		base: '1rem',
      		lg: '1.125rem',
      		xl: '1.25rem',
      		'2xl': '1.5rem',
      		'3xl': '1.875rem',
      		'4xl': '2.25rem',
      		'5xl': '3rem',
      		'6xl': '4rem',
    	},
  	},
	plugins: [
		require('@tailwindcss/forms'),
		require('@tailwindcss/typography')
  	],
}
