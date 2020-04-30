<template>
    <div>
        <img
            :src="
                'http://localhost:8080/api/user/getGravatar?name=' +
                    this.$props.user.name +
                    '&s=256'
            "
            alt=""
        /><br />
        {{ userString }}
    </div>
</template>
<script>
export default {
    props: {
        user: {
            type: Object,
            default: () => {}
        }
    },
    computed: {
        userString() {
            return JSON.stringify(this.$props.user, null, 4)
        }
    }
}
</script>
