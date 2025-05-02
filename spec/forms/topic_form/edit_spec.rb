# typed: false
# frozen_string_literal: true

RSpec.describe TopicForm::Edit, type: :form do
  it "名前が空文字列のとき、エラーになること" do
    form = TopicForm::Edit.new(name: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が `nil` のとき、エラーになること" do
    form = TopicForm::Edit.new(name: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が31文字のとき、エラーになること" do
    form = TopicForm::Edit.new(name: "a" * 31)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name is too long (maximum is 30 characters)")
  end

  it "名前がすでに使われているとき、エラーになること" do
    space = create(:space_record)
    create(:topic_record, space_record: space, name: "すでにあるトピック")
    topic = create(:topic_record, space_record: space)
    form = TopicForm::Edit.new(topic_record: topic, name: "すでにあるトピック")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name has already been taken")
  end

  it "名前が30文字のとき、エラーにならないこと" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(topic_record: topic, name: "a" * 30, description: "test", visibility: "public")

    expect(form).to be_valid
  end

  it "名前が別のスペースで使われているとき、エラーにならないこと" do
    space_1 = create(:space_record)
    space_2 = create(:space_record)
    space_1_topic = create(:topic_record, space_record: space_1)
    create(:topic_record, space_record: space_2, name: "トピック")
    form = TopicForm::Edit.new(topic_record: space_1_topic, name: "トピック", description: "test", visibility: "public")

    expect(form).to be_valid
  end

  it "説明文が151文字のとき、エラーになること" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "a" * 151,
      visibility: "public"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Description is too long (maximum is 150 characters)")
  end

  it "説明文が150文字のとき、エラーにならないこと" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "a" * 150,
      visibility: "public"
    )

    expect(form).to be_valid
  end

  it "説明文が空文字列のとき、エラーにならないこと" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "",
      visibility: "public"
    )

    expect(form).to be_valid
  end

  it "説明文が `nil` のとき、空文字列に変換され、エラーにならないこと" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: nil,
      visibility: "public"
    )

    expect(form).to be_valid
    expect(form.description).to eq("")
  end

  it "公開設定の値が不正のとき、エラーになること" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "test",
      visibility: "invalid_visibility"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Visibility is not included in the list")
  end

  it "公開設定が指定されていないとき、エラーになること" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "test"
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Visibility can't be blank")
  end

  it "公開設定が指定されているとき、エラーにならないこと" do
    topic = create(:topic_record)
    form = TopicForm::Edit.new(
      topic_record: topic,
      name: "テストトピック",
      description: "test",
      visibility: "public"
    )

    expect(form).to be_valid
  end
end
