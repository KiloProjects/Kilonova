<template>
    <div>
        <b-form @submit.prevent="createProblem">
            <b-form-group label="Titlu Problema:" label-for="problemTitle">
                <b-form-input
                    id="problemTitle"
                    v-model="title"
                    type="text"
                    required
                    placeholder="Titlu"
                />
            </b-form-group>
            <b-button type="submit">Trimite</b-button>
        </b-form>
        <!-- <editor v-model="text" /> -->
        <pre>{{ text }}</pre>
    </div>
</template>
<script>
// import Editor from '~/components/Editor'
export default {
    // components: { Editor },
    middleware: 'auth',
    data() {
        return {
            title: null,
            text: null
        }
    },
    methods: {
        async createProblem() {
            try {
                const data = await this.$axios({
                    method: 'POST',
                    url: '/problem/create',
                    params: {
                        title: this.title
                    }
                })
                // eslint-disable-next-line no-console
                console.log(data)
                this.$router.push(`/probleme/${data.data.data}/edit`)
            } catch (e) {
                // eslint-disable-next-line no-console
                console.error(e)
                this.text = e.response.data
            }
        }
    }
}
</script>
