# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in/two_factor/new", type: :request do
  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    get "/sign_in/two_factor/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` がセッションにないとき、ログインページにリダイレクトすること" do
    get "/sign_in/two_factor/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が無効なとき、ログインページにリダイレクトすること" do
    set_session(pending_user_id: "invalid_id")

    get "/sign_in/two_factor/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが無効 & `pending_user_id` が有効なとき、ログインページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    set_session(pending_user_id: user_record.id)

    get "/sign_in/two_factor/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていない & 2FAが有効 & `pending_user_id` が有効なとき、認証コード入力画面が表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    get "/sign_in/two_factor/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("二要素認証")
    expect(response.body).to include("認証コード")
  end
end
