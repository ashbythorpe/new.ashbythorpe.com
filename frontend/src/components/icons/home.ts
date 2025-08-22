import { Component, element, noShadowRoot, staticElement } from "../../custom-elements";

@noShadowRoot
@staticElement
@element("home-icon", "./home.html")
export default class extends Component {};
