# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/settings/general", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    patch "/s/#{space.identifier}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/settings/general"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small, identifier: "space-identifier")
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    expect(space.identifier).to eq("space-identifier")

    patch("/s/#{space.identifier}/settings/general", params: {
      space_form_edit: {
        identifier: "",
        name: ""
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("識別子を入力してください")

    # バリデーションエラーになったのでスペースは更新されていないはず
    expect(space.reload.identifier).to eq("space-identifier")
  end

  it "スペースに参加している & 入力値が正しいとき、スペースが更新できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small, identifier: "space-identifier")
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    expect(space.identifier).to eq("space-identifier")

    patch("/s/#{space.identifier}/settings/general", params: {
      space_form_edit: {
        identifier: "updated-identifier",
        name: "Updated Name"
      }
    })
    space.reload

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/settings/general")

    expect(space.identifier).to eq("updated-identifier")
  end
end
