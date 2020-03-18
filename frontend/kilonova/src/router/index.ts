import Vue from "vue";
import VueRouter from "vue-router";
import Main from "../views/Main.vue";
import Signup from "../views/Signup.vue";
import Login from "../views/Login.vue";
import Profile from "../views/Profile.vue";
import CreateProblem from "../views/CreateProblem.vue";
import Problem from "../views/Problem.vue";

Vue.use(VueRouter);

const routes = [
    {
        path: "/",
        name: "Homepage",
        component: Main,
    },
    {
        path: "/signup",
        name: "Sign Up",
        component: Signup,
    },
    {
        path: "/login",
        name: "Log In",
        component: Login,
    },
    {
        path: "/profile/:name",
        name: "Profile",
        component: Profile,
    },
    {
        path: "/createProblem",
        name: "Create Problem",
        component: CreateProblem,
    },
    {
        path: "/problem/:id",
        component: Problem,
    },
];

const router = new VueRouter({
    routes,
    mode: "history",
});

export default router;
