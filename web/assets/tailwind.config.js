const allcolors = require('tailwindcss/colors');

module.exports = {
//  mode: 'jit',
  darkMode: 'media',
  purge: [
	'../../web/templ/**/*.templ',
	'../../web/templ/*.templ',
  ],
  theme: {
    extend: {},
	  colors: {
		  ...allcolors
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
  variants: {
	extend: {
		textColor: ['active'],
		backgroundColor: ['active'],
		borderColor: ['active'],
		ringWidth: ['hover', 'active'],
		ringColor: ['hover', 'active']
	}
  },
  plugins: [
	require('@tailwindcss/forms'),
	require('@tailwindcss/typography')
  ],
}
