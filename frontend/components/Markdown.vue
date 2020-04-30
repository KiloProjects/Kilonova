<template>
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-html="content"></div>
</template>
<script>
import markdownit from 'markdown-it'
import hljs from 'highlight.js/lib/highlight.js'
hljs.registerLanguage('cpp', require('highlight.js/lib/languages/cpp.js'))
hljs.registerLanguage('python', require('highlight.js/lib/languages/python.js'))
hljs.registerLanguage('go', require('highlight.js/lib/languages/go.js'))

export default {
    props: {
        value: {
            type: String,
            default: ''
        }
    },
    computed: {
        content() {
            const c = markdownit({
                linkify: true,
                injected: true,
                highlight: (str, lang) => {
                    if (lang && hljs.getLanguage(lang)) {
                        try {
                            return hljs.highlight(lang, str).value
                        } catch (__) {}
                        return '' // use external default escaping
                    }
                }
            }).render(this.value)
            return c
        }
    }
}
</script>
