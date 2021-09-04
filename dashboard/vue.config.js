module.exports = {
    devServer:{
        proxy: {
            '/api': {
                target: 'http://yaho.dev:8080/',
                changeOrigin: true
            },
        },
    }
}