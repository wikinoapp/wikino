# typed: false
# frozen_string_literal: true

RSpec.describe "POST /spaces", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    post "/spaces"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    expect(SpaceRecord.count).to eq(0)

    post("/spaces", params: {
      new_space_form: {
        identifier: "", # 識別子が空
        name: "Space Name"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("識別子を入力してください")

    # バリデーションエラーになったのでスペースは作成されていないはず
    expect(SpaceRecord.count).to eq(0)
  end

  it "入力値が正常なとき、スペースが作成できること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    expect(SpaceRecord.count).to eq(0)

    post("/spaces", params: {
      new_space_form: {
        identifier: "test-space",
        name: "テストスペース"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/test-space")

    expect(SpaceRecord.count).to eq(1)
    space = SpaceRecord.first
    expect(space.identifier).to eq("test-space")
    expect(space.name).to eq("テストスペース")
  end
end
