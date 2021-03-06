<template>
  <div class="c-app flex-row align-items-center">
    <CContainer>
      <CRow class="justify-content-center">
        <CCol md="8">
          <CCard>
            <CCardBody>
              <h2>Login</h2>
              <p v-if="errorMsg!=''" class="alert alert-danger">{{ errorMsg }}</p>
              <p v-if="infoMsg!=''" class="text-muted">{{ infoMsg }}</p>
            </CCardBody>
          </CCard>
        </CCol>
      </CRow>
    </CContainer>
  </div>
</template>

<script>
import clientUtils from "@/utils/api_client"
import utils from "@/utils/app_utils"
import appConfig from "@/utils/app_config"

const waitInfoMsg = "Please wait..."
const waitLoginInfoMsg = "Logging in, please wait..."

export default {
  name: 'CheckLogin',
  data() {
    return {
      returnUrl: this.$route.query.returnUrl ? this.$route.query.returnUrl : "/",
      errorMsg: "",
      infoMsg: waitInfoMsg,
      waitCounter: -1,
    }
  },
  mounted() {
    if (this.errorMsg != "") {
      return
    }
    let appId = this.$route.query.app ? this.$route.query.app : appConfig.APP_ID
    this.waitCounter = 0
    this._doWaitMessage()
    let sess = utils.loadLoginSession()
    if (!sess || !sess.uid || !sess.token) {
      this._goLogin(waitInfoMsg, appId, this.returnUrl)
      return
    }
    clientUtils.apiDoPost(clientUtils.apiVerifyLoginToken, {
          token: sess.token,
          app: appId,
          return_url: this.returnUrl,
        },
        (apiRes) => {
          if (apiRes.status != 200) {
            let msg = "Session invalid (code " + apiRes.status + "), redirecting to login page..."
            this._goLogin(msg, appId, this.returnUrl)
            return
          }
          let returnUrl = apiRes.extras.return_url
          this._doSaveLoginSessionAndLogin(apiRes.data, returnUrl)
        },
        (err) => {
          const msg = "Session verification error [" + err + "], redirecting to login page..."
          console.error(msg)
          this._goLogin(msg, appId, this.returnUrl)
        })
  },
  methods: {
    _goLogin(msg, appId, returnUrl) {
      this.errorMsg = ""
      this.infoMsg = msg
      this.waitCounter = -1
      setTimeout(() => {
        this.$router.push({name: "Login", query: {returnUrl: returnUrl, app: appId}})
      }, 2000)
    },
    _doWaitMessage() {
      if (this.waitCounter >= 0) {
        this.waitCounter++
        this.infoMsg = waitLoginInfoMsg + " " + this.waitCounter
        setTimeout(() => {
          this._doWaitMessage()
        }, 2000)
      }
    },
    _doSaveLoginSessionAndLogin(token, returnUrl) {
      this.waitCounter = -1
      const jwt = utils.parseJwt(token)
      utils.saveLoginSession({uid: jwt.payloadObj.uid, token: token})
      window.location.href = returnUrl != "" ? returnUrl : "/"
    },
  }
}
</script>
