<template>
    <div>
        <a :href="'/probleme/' + problem['ID'] + '/edit'">EDIT</a>
        <h3>
            <b-badge>{{ problem['ID'] }}</b-badge>
            {{ problem['title'] }}
        </h3>
        <markdown
            v-if="problem['description']"
            v-model="problem['description']"
        />
        <!-- <pre>{{ JSON.stringify(problem, null, 4) }}</pre> -->
    </div>
</template>
<script>
import Markdown from '~/components/Markdown'
export default {
    components: {
        Markdown
    },
    props: {
        problem: {
            type: Object,
            default: () => {}
        }
    }
}
</script>
