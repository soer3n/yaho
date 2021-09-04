import axios from 'axios'

export default {
  name: 'Dashboard',
  resources: {},
  data () {
    return {
      isCollapse: true,
      resources: {}
    }
  },
  async created () {
    const path = '/api/resources'
    await axios.get(path)
      .then((res) => {
        this.resources = res.data.Data.objects
        console.log(res.data.Data)
      })
      .catch((error) => {
        // eslint-disable-next-line
        console.error(error)
      })
    console.log('update')
  }
}
