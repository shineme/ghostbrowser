import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import './graphite.css'
import './themes-gen.css'
import './themes.css'
import { bootTheme } from './theme'
bootTheme()
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
