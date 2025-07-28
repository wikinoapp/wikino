# typed: false
# frozen_string_literal: true

RSpec.describe Pages::EditForm, type: :form do
  it "SpaceMember が指定されていないとき、エラーになること" do
    form = Pages::EditForm.new

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Space member recordを入力してください")
  end

  it "Page が指定されていないとき、エラーになること" do
    form = Pages::EditForm.new

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Page recordを入力してください")
  end

  it "トピック番号が `nil` のとき、エラーになること" do
    form = Pages::EditForm.new(topic_number: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("トピックを入力してください")
  end

  it "トピック番号に対応するトピックが存在しないとき、エラーになること" do
    form = Pages::EditForm.new(topic_number: 1)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("トピックを入力してください")
  end

  it "タイトルが空文字列のとき、エラーになること" do
    form = Pages::EditForm.new(title: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("タイトルを入力してください")
  end

  it "タイトルが `nil` のとき、エラーになること" do
    form = Pages::EditForm.new(title: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("タイトルを入力してください")
  end

  it "タイトルが201文字のとき、エラーになること" do
    form = Pages::EditForm.new(title: "a" * 201)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("タイトルは200文字以内で入力してください")
  end

  it "タイトルが重複しているとき、エラーになること" do
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)
    space_member_record = create(:space_member_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record:)
    other_page = create(:page_record, topic_record: topic, title: "a")
    page_record = create(:page_record, space_record: space, topic_record: topic)
    form = Pages::EditForm.new(
      title: "a",
      space_member_record:,
      page_record:,
      topic_number: topic.number
    )

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include(
      "タイトルは既に存在しています。<a class=\"underline\" href=\"/s/#{space.identifier}/pages/#{other_page.number}/edit\">既存のページを編集</a>"
    )
  end

  it "タイトルが200文字のとき、エラーにならないこと" do
    form = Pages::EditForm.new(
      title: "a" * 200,
      **valid_attributes.except(:title)
    )

    expect(form).to be_valid
  end

  it "タイトルが別トピックのページと重複しているとき、エラーにならないこと" do
    other_topic = create(:topic_record)
    create(:page_record, topic_record: other_topic, title: "a")
    form = Pages::EditForm.new(
      title: "a",
      **valid_attributes.except(:title)
    )

    expect(form).to be_valid
  end

  it "本文が空文字列のとき、エラーにならないこと" do
    form = Pages::EditForm.new(
      body: "",
      **valid_attributes.except(:body)
    )

    expect(form).to be_valid
  end

  it "本文が `nil` のとき、空文字列に変換され、エラーにならないこと" do
    form = Pages::EditForm.new(
      body: nil,
      **valid_attributes.except(:body)
    )

    expect(form).to be_valid
    expect(form.body).to eq("")
  end

  private def valid_attributes
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)
    space_member_record = create(:space_member_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record:)
    page_record = create(:page_record, space_record: space, topic_record: topic)

    {
      space_member_record:,
      page_record:,
      topic_number: topic.number,
      title: "a",
      body: "a"
    }
  end
end
