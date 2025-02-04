# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/pages/:page_number", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    page = create(:page, space:)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user, :with_password)

    space = create(:space, :small)
    page = create(:page, space:)

    other_space = create(:space)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    create(:space_member, space:, user:)

    topic = create(:topic, space:)
    page = create(:page, space:, topic:, title: "A Page")

    sign_in(user:)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      edit_page_form: {
        topic_number: topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(404)

    # 404になったのでページは更新されていないはず
    expect(page.reload.title).to eq("A Page")
  end

  it "スペースに参加している & ページのトピックに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    space_member = create(:space_member, space:, user:)

    topic = create(:topic, space:)
    create(:topic_membership, space:, topic:, member: space_member)

    page = create(:page, space:, topic:, title: "A Page")

    sign_in(user:)

    expect(page.title).to eq("A Page")

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      edit_page_form: {
        topic_number: topic.number,
        title: "", # タイトルが空
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("タイトルを入力してください")

    # バリデーションエラーになったのでページは更新されていないはず
    expect(page.title).to eq("A Page")
  end

  it "スペースに参加している & ページのトピックに参加している & 入力値が正しいとき、ページが更新できること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    space_member = create(:space_member, space:, user:)

    topic = create(:topic, space:)
    create(:topic_membership, space:, topic:, member: space_member)

    page = create(:page, space:, topic:, title: "A Page")

    sign_in(user:)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      edit_page_form: {
        topic_number: topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/pages/#{page.number}")

    expect(page.reload.title).to eq("Updated Title")
  end
end
