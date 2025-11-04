import { Component, element, noShadowRoot, staticElement } from "../../custom-elements";

@noShadowRoot
@staticElement
@element("mail-icon", "./mail.html")
export default class extends Component {};
