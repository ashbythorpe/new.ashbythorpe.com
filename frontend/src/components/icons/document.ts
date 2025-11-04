import { Component, element, noShadowRoot, staticElement } from "../../custom-elements";

@noShadowRoot
@staticElement
@element("document-icon", "./document.html")
export default class extends Component {};
