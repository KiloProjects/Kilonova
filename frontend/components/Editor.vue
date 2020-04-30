<template>
    <div>
        <client-only placeholder="Loading Editor...">
            <codemirror v-model="value" :options="options"></codemirror>
        </client-only>
    </div>
</template>
<script>
export default {
    props: {
        value: {
            default: '',
            type: String
        },
        language: {
            default: 'text/x-c++src',
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
