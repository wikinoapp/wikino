# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/bulk_restored_pages", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    post "/s/#{space.identifier}/bulk_restored_pages"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/bulk_restored_pages"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "選択したページに問題があるとき、エラーメッセージを表示すること" do
    space = create(:space, :small)
    user = create(:user, :owner, :with_password, space:)
    topic = create(:topic, space:)
    page = create(:page, :trashed, space:, topic:)

    sign_in(user:)

    post("/s/#{space.identifier}/bulk_restored_pages", params: {
      trashed_pages_form: {
        page_ids: [page.id]
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("参加していないトピックのページが含まれているため復元できません")
  end

  it "選択したページに問題がないとき、ページを復元できること" do
    space = create(:space, :small)
    user = create(:user, :owner, :with_password, space:)
    topic = create(:topic, space:)
    page = create(:page, :trashed, space:, topic:)
    create(:topic_membership, space:, topic:, member: user)

    sign_in(user:)

    expect(page.trashed?).to be(true)

    post("/s/#{space.identifier}/bulk_restored_pages", params: {
      trashed_pages_form: {
        page_ids: [page.id]
      }
    })

    expect(response.status).to eq(302)
    expect(page.reload.trashed?).to be(false)
  end
end
