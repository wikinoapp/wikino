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

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    page = create(:page, space:)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "入力値が不正なとき、エラーメッセージを表示すること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:, title: "A Page")
    create(:topic_membership, space:, topic:, member: user)

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

  it "オーナーとしてログインしている & ページのトピックに参加していないとき、ページが更新できること" do
    space = create(:space, :small)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:)
    user = create(:user, :owner, :with_password, space:)

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

  it "オーナーとしてログインしている & ページのトピックに参加しているとき、ページが更新できること" do
    space = create(:space, :small)
    user = create(:user, :owner, :with_password, space:)
    topic = create(:topic, space:)
    page = create(:page, space:, topic:)
    create(:topic_membership, space:, topic:, member: user)

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
