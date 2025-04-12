# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/settings/exports", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    post "/s/#{space.identifier}/settings/exports"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user, :with_password)

    space = create(:space, :small)

    other_space = create(:space)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    post "/s/#{space.identifier}/settings/exports"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、エクスポートが作成できること" do
    user = create(:user, :with_password)
    space = create(:space, :small, identifier: "space-identifier")
    create(:space_member, space:, user:)

    sign_in(user:)

    expect(Export.count).to eq(0)

    post("/s/#{space.identifier}/settings/exports")

    expect(space.exports.count).to eq(1)
    export = space.exports.first

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/settings/exports/#{export.id}")
  end
end
