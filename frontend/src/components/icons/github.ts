import { Component, element, noShadowRoot, staticElement } from "../../custom-elements";

@noShadowRoot
@staticElement
@element("github-icon", "./github.html")
export default class extends Component {};
