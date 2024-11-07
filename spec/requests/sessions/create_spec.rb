# typed: false
# frozen_string_literal: true

RSpec.describe "POST /sessions", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)
    sign_in(user:)

    # ログインしているのでセッションは1つ
    expect(Session.count).to eq(1)

    post("/sessions", params: {
      session_form: {
        space_identifier: user.space.identifier,
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))

    # ログインしているのでセッションは増えないはず
    expect(Session.count).to eq(1)
  end

  it "ログインしている & `skip_no_authentication` が付与されているときはログインできること" do
    user = create(:user, :with_password)
    sign_in(user:)

    # ログインしているのでセッションは1つ
    expect(Session.count).to eq(1)

    post("/sessions?skip_no_authentication=true", params: {
      session_form: {
        space_identifier: user.space.identifier,
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))

    # もう一度ログインし直すのでセッションは2つになるはず
    expect(Session.count).to eq(2)
  end

  it "ログインしていないときはログインできること" do
    # ログインしていないのでセッションはまだ無い
    expect(Session.count).to eq(0)

    user = create(:user, :with_password)

    post("/sessions", params: {
      session_form: {
        space_identifier: user.space.identifier,
        email: user.email,
        password: "passw0rd"
      }
    })
    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))

    # ログインしたのでセッションが1つ生まれるはず
    expect(Session.count).to eq(1)
  end
end
