# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::EditForm, type: :form do
  it "Space が指定されていないとき、エラーになること" do
    form = Spaces::EditForm.new

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Space recordを入力してください")
  end

  it "識別子が空文字列のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子を入力してください")
  end

  it "識別子が `nil` のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子を入力してください")
  end

  it "識別子が21文字のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "a" * 21)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子は20文字以内で入力してください")
  end

  it "識別子が予約語のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "www")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子は使用できません")
  end

  it "識別子の形式が不正なとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "a@b")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子は不正な値です")
  end

  it "識別子がすでに使われているとき、エラーになること" do
    create(:space_record, identifier: "a")
    space = create(:space_record)
    form = Spaces::EditForm.new(space_record: space, identifier: "a")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("識別子は既に存在しています")
  end

  it "名前が空文字列のとき、エラーになること" do
    form = Spaces::EditForm.new(name: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("名前を入力してください")
  end

  it "名前が `nil` のとき、エラーになること" do
    form = Spaces::EditForm.new(name: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("名前を入力してください")
  end

  it "名前が31文字のとき、エラーになること" do
    form = Spaces::EditForm.new(name: "a" * 31)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("名前は30文字以内で入力してください")
  end
end
