module.exports = {
  future: {
    removeDeprecatedGapUtilities: true,
    purgeLayersByDefault: true,
  },
  purge: [
	'../../web/templ/**/*.templ',
	'../../web/templ/*.templ'
  ],
  theme: {
    extend: {},
  },
  variants: {},
  plugins: [
	require('@tailwindcss/ui')
  ],
}
