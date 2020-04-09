<template>
    <div>{{ JSON.stringify(problem, null, 4) }}</div>
</template>
<script>
export default {
    props: {
        problem: {
            type: Object,
            default: () => {}
        }
    }
}
</script>
