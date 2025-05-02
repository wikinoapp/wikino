# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/pages/:page_number/draft_page", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    draft_page = create(:draft_page_record, space_record: space)

    patch "/s/#{space.identifier}/pages/#{draft_page.page_record.number}/draft_page"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404を返すこと" do
    space = create(:space_record, :small)
    draft_page = create(:draft_page_record, space_record: space)
    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/pages/#{draft_page.page_record.number}/draft_page"

    expect(response.status).to eq(404)
  end

  it "ページのトピックに参加していないとき、404を返すこと" do
    space = create(:space_record, :small)
    page = create(:page_record, :published, space_record: space)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}/draft_page", params: {
      page_form_edit: {
        topic_number: page.topic_record.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(404)
  end

  it "ページのトピックに参加しているとき、下書きページが更新できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, space_record: space)
    page = create(:page_record, :published, space_record: space, topic_record: topic)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}/draft_page", params: {
      page_form_edit: {
        topic_number: page.topic_record.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(200)
    expect(response.body).to include("下書き保存")
  end
end
