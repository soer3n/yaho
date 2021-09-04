import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import Home from '../views/Home/Home.vue'
import Select from '../views/Select/Select.vue'
import Navigation from '../views/Navigation/Navigation.vue'

const Installation = {
  template: '<div>Installation</div>'
}

const BasicUsage = {
  template: '<div>basic-usage</div>'
}

const SubPage1 = {
  template: '<div>SubPage1</div>'
}

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'Select',
    component: Select
  },
  {
    path: '/charts/sublink',
    name: 'Box',
    component: Navigation
  }
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes
})

console.log('history', process.env.BASE_URL)

export default router
