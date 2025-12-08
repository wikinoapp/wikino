# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /settings/profile", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    patch "/settings/profile"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & 入力値が不正なとき、エラーメッセージを表示すること" do
    user_record = create(:user_record, :with_password, name: "Before Name", description: "Before Description")

    sign_in(user_record:)

    expect(user_record.name).to eq("Before Name")
    expect(user_record.description).to eq("Before Description")

    patch("/settings/profile", params: {
      profiles_edit_form: {
        atname: "", # アットネームが空
        name: "Updated Name",
        description: "Updated Description"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("アットネームを入力してください")

    # バリデーションエラーになったのでプロフィールは更新されていないはず
    expect(user_record.reload.name).to eq("Before Name")
    expect(user_record.description).to eq("Before Description")
  end

  it "ログインしている & 入力値が正しいとき、プロフィールが更新できること" do
    user_record = create(:user_record, :with_password, atname: "before_atname", name: "Before Name", description: "Before Description")

    sign_in(user_record:)

    expect(user_record.name).to eq("Before Name")
    expect(user_record.description).to eq("Before Description")

    patch("/settings/profile", params: {
      profiles_edit_form: {
        atname: "updated_atname",
        name: "Updated Name",
        description: "Updated Description"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/settings/profile")

    expect(user_record.reload.atname).to eq("updated_atname")
    expect(user_record.name).to eq("Updated Name")
    expect(user_record.description).to eq("Updated Description")
  end
end
