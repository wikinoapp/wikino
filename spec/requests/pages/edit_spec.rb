# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/pages/:page_number/edit", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, space:)
    page = create(:page_record, space:, topic:, title: "ページタイトル")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加しているとき、編集ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, space:)
    page = create(:page_record, space:, topic:, title: "ページタイトル")
    create(:topic_member_record, space:, topic:, space_member:)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/pages/#{page.number}/edit"

    expect(response.status).to eq(200)
    expect(response.body).to include("ページタイトル")
  end
end
