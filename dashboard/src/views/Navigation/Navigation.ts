
export default {
  name: 'Box',
  data () {
    return {
      fixedStatus: {
        headerIsFixed: false
      },
      propsData: { ...createData() },
      formData: { ...createData() }
    }
  }
}

const createData = () => ({
  threshold: 0,
  headerClass: 'vue-fixed-header',
  fixedClass: 'vue-fixed-header--isFixed',
  hideScrollUp: false
})
