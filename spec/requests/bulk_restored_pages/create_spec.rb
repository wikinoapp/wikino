# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/bulk_restored_pages", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    post "/s/#{space.identifier}/bulk_restored_pages"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404を返すこと" do
    space = create(:space_record, :small)
    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/bulk_restored_pages"

    expect(response.status).to eq(404)
  end

  it "選択したページに問題があるとき、エラーメッセージを表示すること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, space_record: space) # このトピックに参加していない
    page = create(:page_record, :trashed, space_record: space, topic_record: topic)

    sign_in(user_record: user)

    post("/s/#{space.identifier}/bulk_restored_pages", params: {
      page_form_bulk_restoring: {
        page_ids: [page.id]
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("参加していないトピックのページが含まれているため復元できません")
  end

  it "選択したページに問題がないとき、ページを復元できること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    space_member = create(:space_member_record, space_record: space, user_record: user)
    topic = create(:topic_record, space_record: space)
    page = create(:page_record, :trashed, space_record: space, topic_record: topic)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    sign_in(user_record: user)

    expect(page.trashed?).to be(true)

    post("/s/#{space.identifier}/bulk_restored_pages", params: {
      page_form_bulk_restoring: {
        page_ids: [page.id]
      }
    })

    expect(response.status).to eq(302)
    expect(page.reload.trashed?).to be(false)
  end
end
