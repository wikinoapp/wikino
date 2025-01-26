# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/trash", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    page = create(:page, space:)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、403を返すこと" do
    space = create(:space, :small)
    page = create(:page, space:)

    other_space = create(:space)
    user = create(:user, :with_password)
    create(:space_member, :owner, space: other_space, user:)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(403)
  end

  it "指定したページが存在しないとき、エラーメッセージを表示すること" do
    space = create(:space, :small)
    user = create(:user, :with_password)
    create(:space_member, :owner, space:, user:)

    sign_in(user:)

    post "/s/#{space.identifier}/pages/0/trash"

    expect(response.status).to eq(404)
  end

  it "オーナーとしてログインしているとき、ゴミ箱に移動できること" do
    space = create(:space, :small)
    user = create(:user, :with_password)
    space_member = create(:space_member, :owner, space:, user:)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:)
    create(:topic_membership, space:, topic:, member: space_member)

    sign_in(user:)

    expect(page.trashed?).to be(false)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(302)
    expect(page.reload.trashed?).to be(true)
  end
end
