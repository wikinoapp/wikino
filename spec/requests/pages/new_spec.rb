# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/pages/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    topic = create(:topic_record, :public, space_record: space)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    space = create(:space_record, :small)
    topic = create(:topic_record, :public, space_record: space)

    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, :public, space_record: space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加しているとき、ページを作成してから編集ページにリダイレクトすること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, :public, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    expect(PageRecord.count).to eq(0)

    get "/s/#{space.identifier}/topics/#{topic.number}/pages/new"

    expect(response.status).to eq(302)

    expect(PageRecord.count).to eq(1)
    page = topic.pages.first

    expect(response).to redirect_to("/s/#{space.identifier}/pages/#{page.number}/edit")
  end
end
