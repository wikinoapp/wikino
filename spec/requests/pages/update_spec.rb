# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/pages/:page_number", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
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
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    expect(page.title).to eq("A Page")

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
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
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
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
