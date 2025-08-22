import { Component, element, noShadowRoot, staticElement } from "../../custom-elements";

@noShadowRoot
@staticElement
@element("info-icon", "./info.html")
export default class extends Component {};
