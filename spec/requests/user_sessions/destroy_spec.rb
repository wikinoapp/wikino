# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /user_session", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトされること" do
    delete("/user_session")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしているとき、ログアウトできること" do
    user = create(:user_record, :with_password)
    sign_in(user_record: user)

    # ログインしているのでセッションは1つ
    expect(UserSessionRecord.count).to eq(1)

    delete("/user_session")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # ログアウトしたのでセッションは削除されているはず
    expect(UserSessionRecord.count).to eq(0)
  end
end
