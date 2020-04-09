<template>
    <div>
        TEST 123
        <editor />
    </div>
</template>
<script>
import Editor from '~/components/Editor'
export default {
    components: { editor: Editor },
    data() {
        return {
            text: null
        }
    }
}
</script>
