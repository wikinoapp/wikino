# typed: false
# frozen_string_literal: true

RSpec.describe "GET /@:atname", type: :request do
  it "存在しないプロフィールにアクセスしたとき、404を返すこと" do
    get "/@non_existent_atname"

    expect(response.status).to eq(404)
  end

  it "ログインしていないとき、ページが表示されること" do
    user = create(:user_record)

    get "/@#{user.atname}"

    expect(response.status).to eq(200)
    expect(response.body).to include(user.atname)
  end

  it "ログインしている & 自分のプロフィールのとき、ページが表示されること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    get "/@#{user.atname}"

    expect(response.status).to eq(200)
    expect(response.body).to include(user.atname)
  end

  it "ログインしている & 他人のプロフィールのとき、ページが表示されること" do
    user = create(:user_record, :with_password)
    other_user = create(:user_record)

    sign_in(user_record: user)

    get "/@#{other_user.atname}"

    expect(response.status).to eq(200)
    expect(response.body).to include(other_user.atname)
  end
end
