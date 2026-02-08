# typed: false
# frozen_string_literal: true

RSpec.describe "POST /user_session", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)
    sign_in(user_record: user)

    # ログインしているのでセッションは1つ
    expect(UserSessionRecord.count).to eq(1)

    post("/user_session", params: {
      user_sessions_creation_form: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # ログインしているのでセッションは増えないはず
    expect(UserSessionRecord.count).to eq(1)
  end
end
