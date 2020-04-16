<template>
    <div>
        <a :href="'/probleme/' + problem['ID'] + '/edit'">EDIT</a>
        <h3>
            <b-badge>{{ problem['ID'] }}</b-badge>
            {{ problem['name'] }}
        </h3>
        <pre>
            {{ problem['text'] }}
        </pre>
        <pre>{{ JSON.stringify(problem, null, 4) }}</pre>
    </div>
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
