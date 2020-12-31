import { LitElement, html, css, CSSResult, TemplateResult } from "lit-element";
import { USER_PROFILE_ACTOR_ADDRESS } from "../constants";
import ActorStore from "../actors/actorStore";
import { UserActor } from "../actors/user";
import "./login-form";
import "./register-form";
import "@material/mwc-top-app-bar-fixed";
import "@material/mwc-icon-button";
import "@material/mwc-menu";
import "@material/mwc-list/mwc-list-item";
import "@material/mwc-icon";

class TopBar extends LitElement {
  showRegister: Boolean = false;
  showUserProfile: Boolean = false;
  snackMessage: String = "";
  username: String = "";
  email: String = "";

  static get styles(): CSSResult {
    return css`
      login-container {
        display: grid;
        justify-content: center;
        margin-top: 10vh;
      }
    `;
  }

  async updated(): Promise<void> {
    if (this.showUserProfile) return;

    const userActor: any = ActorStore.lookup(USER_PROFILE_ACTOR_ADDRESS);
    const userProfile = await userActor.message(UserActor.MessageTypes.GetUser);

    if (!userProfile) return;

    this.username = userProfile.username;
    this.email = userProfile.email;
    this.showUserProfile = true;
    this.requestUpdate();
  }

  handleSnackbarMsg(evt: Event & { detail: String }): void {
    const snackbar:
      | (HTMLElement & { show: Function })
      | null
      | undefined = this.shadowRoot?.querySelector("#loginMessage");
    if (!snackbar) return console.error("no snackbar");

    this.snackMessage = evt.detail;

    this.requestUpdate();
    snackbar.show();
  }

  handleRegisterBtn(): void {
    this.showRegister = !this.showRegister;
    this.requestUpdate();
  }

  handleLogout(): void {}

  handleProfileClick(): void {
    const profileBtn:
      | HTMLElement
      | null
      | undefined = this.shadowRoot?.querySelector("#profileBtn");
    const menu:
      | (HTMLElement & { anchor: HTMLElement; show: Function })
      | null
      | undefined = this.shadowRoot?.querySelector("#menu");

    if (!profileBtn) return console.error("profile btn doesn't exist");
    if (!menu) return console.error("menu element doesn't exist");

    menu.anchor = profileBtn;
    menu.show();
  }

  render(): TemplateResult {
    const login = html`<login-form
        @control-changed="${this.handleSnackbarMsg}"
      ></login-form>
      <mwc-list-item @click=${this.handleRegisterBtn}>
        <mwc-button label="Register"></mwc-button>
      </mwc-list-item> `;
    let output: TemplateResult = login;
    let loginform = html``;

    if (this.showRegister) {
      output = html`<register-form
        @control-changed="${this.handleSnackbarMsg}"
      ></register-form>`;
    }

    if (this.showUserProfile) {
      output = html`
        <mwc-list-item>
          <mwc-icon slot="graphic">person</mwc-icon>
          ${this.username}</mwc-list-item
        >
        <mwc-list-item>${this.email}</mwc-list-item>
        <mwc-list-item @click=${this.handleLogout}>
          <mwc-button label="Logout"></mwc-button>
        </mwc-list-item>
      `;
    } else {
      loginform = html`<login-container> ${output} </login-container>`;
    }

    return html`
      <mwc-top-app-bar-fixed>
        <mwc-icon-button icon="menu" slot="navigationIcon"></mwc-icon-button>
        <div slot="title">Member Dashboard</div>
        <div slot="actionItems">${this.username}</div>
        <mwc-icon-button
          id="profileBtn"
          @click=${this.handleProfileClick}
          icon="person"
          slot="actionItems"
        ></mwc-icon-button>
        <mwc-menu id="menu" activatable> ${output} </mwc-menu>

        ${loginform}

        <mwc-snackbar id="loginMessage" stacked labelText=${this.snackMessage}>
        </mwc-snackbar>
      </mwc-top-app-bar-fixed>
    `;
  }
}

customElements.define("top-bar", TopBar);
