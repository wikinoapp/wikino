# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /s/:space_identifier/session", type: :request do
  it "ログインしているとき、ログアウトできること" do
    user = create(:user, :with_password)
    sign_in(user:)

    # ログインしているのでセッションは1つ
    expect(Session.count).to eq(1)

    delete("/s/#{user.space.identifier}/session")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/")

    # ログアウトしたのでセッションは削除されているはず
    expect(Session.count).to eq(0)
  end

  it "ログインしていないときはログインページにリダイレクトされること" do
    user = create(:user, :with_password)

    delete("/s/#{user.space.identifier}/session")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
