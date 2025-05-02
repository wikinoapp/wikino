# typed: false
# frozen_string_literal: true

RSpec.describe "POST /user_session", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user_record, :with_password)
    sign_in(user_record: user)

    # ログインしているのでセッションは1つ
    expect(UserSessionRecord.count).to eq(1)

    post("/user_session", params: {
      user_session_form_creation: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # ログインしているのでセッションは増えないはず
    expect(UserSessionRecord.count).to eq(1)
  end

  it "ログインしていないときはログインできること" do
    # ログインしていないのでセッションはまだ無い
    expect(UserSessionRecord.count).to eq(0)

    user = create(:user_record, :with_password)

    post("/user_session", params: {
      user_session_form_creation: {
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")

    # ログインしたのでセッションが1つ生まれるはず
    expect(UserSessionRecord.count).to eq(1)
  end

  it "入力値が間違っているときはログインできないこと" do
    # ログインしていないのでセッションはまだ無い
    expect(UserSessionRecord.count).to eq(0)

    user = create(:user_record, :with_password)

    post("/user_session", params: {
      user_session_form_creation: {
        email: user.email,
        password: "password" # パスワードを間違えている
      }
    })
    expect(response.status).to eq(422)
    expect(response.body).to include("ログインに失敗しました")

    # ログインに失敗したのでセッションは作られていないはず
    expect(UserSessionRecord.count).to eq(0)
  end
end
