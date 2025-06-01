# typed: false
# frozen_string_literal: true

RSpec.describe "GET /user_session/two_factor_auth/new", type: :request do
  it "pending_user_idがセッションにないとき、ログインページにリダイレクトすること" do
    get "/user_session/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "pending_user_idが無効なとき、ログインページにリダイレクトすること" do
    set_session(pending_user_id: "invalid_id")

    get "/user_session/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ユーザーの2FAが無効なとき、ログインページにリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    set_session(pending_user_id: user_record.id)

    get "/user_session/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ユーザーの2FAが有効なとき、認証コード入力画面が表示されること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)
    set_session(pending_user_id: user_record.id)

    get "/user_session/two_factor_auth/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("二要素認証")
    expect(response.body).to include("認証コード")
  end

  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)
    create(:user_two_factor_auth_record, :enabled, user_record:)

    sign_in_with_2fa(user_record:)

    get "/user_session/two_factor_auth/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end
end