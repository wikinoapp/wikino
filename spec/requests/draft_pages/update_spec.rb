# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/pages/:page_number/draft_page", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    draft_page = create(:draft_page, space:)

    patch "/s/#{space.identifier}/pages/#{draft_page.page.number}/draft_page"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    draft_page = create(:draft_page, space:)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    patch "/s/#{space.identifier}/pages/#{draft_page.page.number}/draft_page"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "オーナーとしてログインしている & ページのトピックに参加していないとき、下書きページが更新できること" do
    space = create(:space, :small)
    page = create(:page, :published, space:)
    user = create(:user, :owner, :with_password, space:)

    sign_in(user:)

    patch("/s/#{space.identifier}/pages/#{page.number}/draft_page", params: {
      edit_page_form: {
        topic_number: page.topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(200)
    expect(response.body).to include("下書き保存")
  end

  it "オーナーとしてログインしている & ページのトピックに参加しているとき、下書きページが更新できること" do
    space = create(:space, :small)
    user = create(:user, :owner, :with_password, space:)
    topic = create(:topic, space:)
    page = create(:page, :published, space:)
    create(:topic_membership, space:, topic:, member: user)

    sign_in(user:)

    patch("/s/#{space.identifier}/pages/#{page.number}/draft_page", params: {
      edit_page_form: {
        topic_number: page.topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(200)
    expect(response.body).to include("下書き保存")
  end
end
