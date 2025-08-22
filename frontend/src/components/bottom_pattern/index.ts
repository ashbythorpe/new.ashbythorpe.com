import { Component, element, noShadowRoot, staticElement } from "../../custom-elements.ts";

@noShadowRoot
@staticElement
@element("bottom-pattern", "./index.html")
export default class extends Component {}
