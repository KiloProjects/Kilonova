export default {
    mode: 'universal',
    server: {
        host: '0.0.0.0'
    },
    /*
     ** Headers of the page
     */
    head: {
        title: process.env.npm_package_name || '',
        meta: [
            { charset: 'utf-8' },
            {
                name: 'viewport',
                content: 'width=device-width, initial-scale=1'
            },
            {
                hid: 'description',
                name: 'description',
                content: process.env.npm_package_description || ''
            }
        ],
        link: [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }]
    },
    /*
     ** Customize the progress-bar color
     */
    loading: { color: '#fff' },
    /*
     ** Global CSS
     */
    css: [
        'codemirror/lib/codemirror.css',
        'codemirror/theme/monokai.css',
        'codemirror/addon/fold/foldgutter.css'
    ],
    /*
     ** Plugins to load before mounting the App
     */
    plugins: [{ src: '~/plugins/codemirror.js', ssr: false }],
    /*
     ** Nuxt.js dev-modules
     */
    buildModules: [
        // Doc: https://github.com/nuxt-community/eslint-module
        '@nuxtjs/eslint-module'
    ],
    /*
     ** Nuxt.js modules
     */
    modules: [
        // Doc: https://bootstrap-vue.js.org
        'bootstrap-vue/nuxt',
        // Doc: https://axios.nuxtjs.org/usage
        '@nuxtjs/axios',
        // Doc: https://auth.nuxtjs.org
        '@nuxtjs/auth'
    ],
    /*
     ** Axios module configuration
     ** See https://axios.nuxtjs.org/options
     */
    axios: {
        baseURL: 'http://kilonova:8080/api',
        browserBaseURL: 'http://localhost:8080/api',
        credentials: true
    },
    /*
     ** Auth configuration
     */
    auth: {
        strategies: {
            local: {
                endpoints: {
                    login: {
                        url: '/auth/login',
                        method: 'post',
                        propertyName: 'data'
                    },
                    user: {
                        url: '/user/getSelf',
                        method: 'get',
                        propertyName: 'data'
                    },
                    logout: {
                        url: '/auth/logout',
                        method: 'post'
                    }
                }
            }
        },
        cookie: {
            prefix: 'auth.',
            options: {
                path: '/'
            }
        },
        token: {
            prefix: '_token.'
        }
    },
    /*
     ** Build configuration
     */
    build: {
        /*
         ** You can extend webpack config here
         */
        extend(config, ctx) {}
    }
}
