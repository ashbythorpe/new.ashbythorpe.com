import './style.css'
import { Component, element } from './custom-elements.ts'
import "./components/navbar/index.ts"

// document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
//   <div>
//     <a href="https://vite.dev" target="_blank">
//       <img src="${viteLogo}" class="logo" alt="Vite logo" />
//     </a>
//     <a href="https://www.typescriptlang.org/" target="_blank">
//       <img src="${typescriptLogo}" class="logo vanilla" alt="TypeScript logo" />
//     </a>
//     <h1>Vite + TypeScript</h1>
//     <div class="card">
//       <button id="counter" type="button"></button>
//     </div>
//     <p class="read-the-docs">
//       Click on the Vite and TypeScript logos to learn more
//     </p>
//   </div>
// `

@element("my-component", "./components/component.html")
export class _ extends Component {};

@element("other-component", "other-component.html")
export class OtherComponent extends Component {};

//
// setupCounter(document.querySelector<HTMLButtonElement>('#counter')!)
