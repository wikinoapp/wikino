# typed: false
# frozen_string_literal: true

RSpec.describe "POST /sessions", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)
    sign_in(user:)

    # ログインしているのでセッションは1つ
    expect(UserSession.count).to eq(1)

    post("/user_sessions", params: {
      user_session_form: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # ログインしているのでセッションは増えないはず
    expect(UserSession.count).to eq(1)
  end

  it "ログインしている & `skip_no_authentication` が付与されているときはログインできること" do
    user = create(:user, :with_password)
    sign_in(user:)

    # ログインしているのでセッションは1つ
    expect(UserSession.count).to eq(1)

    post("/user_sessions?skip_no_authentication=true", params: {
      user_session_form: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # もう一度ログインし直すのでセッションは2つになるはず
    expect(UserSession.count).to eq(2)
  end

  it "ログインしていないときはログインできること" do
    # ログインしていないのでセッションはまだ無い
    expect(UserSession.count).to eq(0)

    user = create(:user, :with_password)

    post("/user_sessions", params: {
      user_session_form: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # ログインしたのでセッションが1つ生まれるはず
    expect(UserSession.count).to eq(1)
  end

  it "入力値が間違っているときはログインできないこと" do
    # ログインしていないのでセッションはまだ無い
    expect(UserSession.count).to eq(0)

    user = create(:user, :with_password)

    post("/user_sessions", params: {
      user_session_form: {
        email: user.email,
        password: "password" # パスワードを間違えている
      }
    })
    expect(response.status).to eq(422)
    expect(response.body).to include("ログインに失敗しました")

    # ログインに失敗したのでセッションは作られていないはず
    expect(UserSession.count).to eq(0)
  end
end
