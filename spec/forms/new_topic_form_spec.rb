# typed: false
# frozen_string_literal: true

RSpec.describe NewTopicForm, type: :form do
  it "名前が空文字列のとき、エラーになること" do
    form = NewTopicForm.new(name: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が `nil` のとき、エラーになること" do
    form = NewTopicForm.new(name: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が31文字のとき、エラーになること" do
    form = NewTopicForm.new(name: "a" * 31)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name is too long (maximum is 30 characters)")
  end

  it "名前がすでに使われているとき、エラーになること" do
    space = create(:space)
    topic = create(:topic, space:)
    form = NewTopicForm.new(space:, name: topic.name)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name has already been taken")
  end

  it "名前が30文字のとき、エラーにならないこと" do
    space = create(:space)
    form = NewTopicForm.new(space:, name: "a" * 30, description: "test", visibility: "public")

    expect(form).to be_valid
  end

  it "名前が別のスペースで使われているとき、エラーにならないこと" do
    space_1 = create(:space)
    space_2 = create(:space)
    create(:topic, space: space_2, name: "トピック")
    form = NewTopicForm.new(space: space_1, name: "トピック", description: "test", visibility: "public")

    expect(form).to be_valid
  end

  it "説明文が151文字のとき、エラーになること" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "a" * 151,
      visibility: "public"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Description is too long (maximum is 150 characters)")
  end

  it "説明文が150文字のとき、エラーにならないこと" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "a" * 150,
      visibility: "public"
    )

    expect(form).to be_valid
  end

  it "説明文が空文字列のとき、エラーにならないこと" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "",
      visibility: "public"
    )

    expect(form).to be_valid
  end

  it "説明文が `nil` のとき、空文字列に変換され、エラーにならないこと" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: nil,
      visibility: "public"
    )

    expect(form).to be_valid
    expect(form.description).to eq("")
  end

  it "公開設定の値が不正のとき、エラーになること" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "test",
      visibility: "invalid_visibility"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Visibility is not included in the list")
  end

  it "公開設定が指定されていないとき、エラーになること" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "test"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Visibility can't be blank")
  end

  it "公開設定が指定されているとき、エラーにならないこと" do
    form = NewTopicForm.new(
      space: create(:space),
      name: "テストトピック",
      description: "test",
      visibility: "public"
    )

    expect(form).to be_valid
  end
end
