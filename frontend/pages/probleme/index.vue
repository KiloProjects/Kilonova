<template>
    <div>
        <ul v-if="probleme">
            <li v-for="problema in probleme" :key="problema.ID">
                <nuxt-link :to="`/probleme/${problema.ID}`">
                    <b-badge> #{{ problema['ID'] }}</b-badge>
                    {{ problema['title'] }}
                </nuxt-link>
            </li>
        </ul>
        <ErrorCard :err="error" />
    </div>
</template>
<script>
import ErrorCard from '~/components/ErrorCard'
export default {
    components: {
        ErrorCard
    },
    async asyncData({ $axios }) {
        try {
            const probleme = await $axios.get('/problem/getAll')
            return {
                probleme: probleme.data.data
            }
        } catch (err) {
            return {
                error: err.response.data
            }
        }
    },
    data() {
        return {
            probleme: null,
            error: null
        }
    }
}
</script>
