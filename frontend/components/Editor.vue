<template>
    <div>
        <no-ssr placeholder="Loading...">
            <codemirror v-model="value" :options="options"></codemirror>
        </no-ssr>
    </div>
</template>
<script>
export default {
    props: {
        value: {
            default: '',
            type: String
        }
    },
    data() {
        return {
            options: {
                theme: 'monokai',
                mode: 'text/x-c++src',
                tabSize: 4,
                indentUnit: 4,
                lineNumbers: true,
                line: true,
                keyMap: 'sublime',
                matchBrackets: true,
                autoCloseBrackets: true,
                styleActiveLine: true
            }
        }
    },
    watch: {
        value(newCode) {
            this.$emit('input', newCode)
        }
    }
}
</script>
