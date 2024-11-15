# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/users/:user_id/account_switch", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)

    post "/s/#{space.identifier}/users/#{user.id}/account_switch"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/users/#{user.id}/account_switch"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしていないユーザーに切り替えようとしたとき、404を返すこと" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    other_space = create(:space, :small)
    other_user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/users/#{other_user.id}/account_switch"

    expect(response.status).to eq(404)
  end

  it "ログインしているユーザーに切り替えできること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    other_space = create(:space, :small)
    other_user = create(:user, :with_password, space: other_space)

    sign_in(user:)
    sign_in(user: other_user)

    expect(Session.count).to eq(2)

    post "/s/#{space.identifier}/users/#{user.id}/account_switch"

    expect(response.status).to eq(302)
    expect(Session.count).to eq(3)
    expect(response).to redirect_to("/s/#{space.identifier}")
  end
end
