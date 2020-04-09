<template>
    <div>
        <Profile v-if="profil" :user="profil" />
        <ErrorCard :err="error" />
    </div>
</template>
<script>
import ErrorCard from '~/components/ErrorCard'
import Profile from '~/components/Profile'
export default {
    components: {
        ErrorCard,
        Profile
    },
    data() {
        return {
            profil: null,
            error: null
        }
    },
    async created() {
        try {
            const data = await this.$axios.get(
                `/user/getByName?name=${this.$route.params.name}`
            )
            this.profil = data.data
        } catch (err) {
            this.error = err.response.data
        }
    }
}
</script>
