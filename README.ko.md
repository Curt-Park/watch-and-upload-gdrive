# Watch and Upload to Google Drive (WUG)

디렉토리를 감시하고 새로운 파일을 Google Drive에 자동으로 업로드하는 Go 기반 CLI 도구입니다.

**English**: See [README.md](README.md) for English documentation.

## 기능

- 실시간 디렉토리 모니터링
- 자동 파일 업로드
- 파일 확장자 필터링 지원
- Google Drive OAuth2 인증

## 설치

### 방법 1: 사전 빌드된 바이너리 다운로드 (권장)

다음 명령어 중 플랫폼에 맞는 것을 사용하여 최신 릴리즈를 다운로드하세요:

#### Linux
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-linux-amd64 -O wug
chmod +x wug
```

#### macOS (Intel)
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-darwin-amd64 -O wug
chmod +x wug
```

#### macOS (Apple Silicon)
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-darwin-arm64 -O wug
chmod +x wug
```

#### Windows
```bash
# PowerShell 사용
Invoke-WebRequest -Uri https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-windows-amd64.exe -OutFile wug.exe

# 또는 curl 사용 (사용 가능한 경우)
curl -L -o wug.exe https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-windows-amd64.exe
```

**참고**: `wget`이 없는 경우 `curl`을 사용할 수 있습니다:
```bash
# Linux/macOS에서 curl 사용
curl -L -o wug https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-linux-amd64
chmod +x wug
```

#### PATH에 설치 (선택사항)
```bash
# Linux/macOS
sudo mv wug /usr/local/bin/wug

# 또는 현재 디렉토리에서 직접 사용
./wug /path/to/directory
```

### 방법 2: 소스에서 빌드

1. 저장소 클론:
```bash
git clone https://github.com/Curt-Park/watch-and-upload-gdrive.git
cd watch-and-upload-gdrive
```

2. 의존성 설치:
```bash
go mod download
```

3. 빌드:
```bash
go build -o wug
```

또는 Makefile 사용:
```bash
make build
```

## 설정

### 1단계: Google Cloud 프로젝트 생성

1. [Google Cloud Console](https://console.cloud.google.com/)로 이동
2. 페이지 상단의 프로젝트 드롭다운 클릭
3. **"새 프로젝트"** 클릭
4. 프로젝트 이름 입력 (예: "wug-drive-uploader")
5. **"만들기"** 클릭
6. 프로젝트가 생성될 때까지 기다린 후 선택

### 2단계: Google Drive API 활성화

1. Google Cloud Console에서 **"API 및 서비스"** > **"라이브러리"**로 이동
2. **"Google Drive API"** 검색
3. 결과에서 **"Google Drive API"** 클릭
4. **"사용 설정"** 버튼 클릭
5. API가 활성화될 때까지 대기 (몇 초 소요될 수 있음)

### 3단계: OAuth 2.0 인증 정보 생성

1. **"API 및 서비스"** > **"사용자 인증 정보"**로 이동
2. 페이지 상단의 **"+ 사용자 인증 정보 만들기"** 클릭
3. 드롭다운 메뉴에서 **"OAuth 클라이언트 ID"** 선택

   **참고**: 처음으로 인증 정보를 만드는 경우, 먼저 OAuth 동의 화면을 구성해야 합니다:
   - **"동의 화면 구성"** 클릭
   - **"외부"** 선택 (Google Workspace 계정이 아닌 경우)
   - **"만들기"** 클릭
   - 필수 정보 입력:
     - **앱 이름**: 이름 입력 (예: "WUG Drive Uploader")
     - **사용자 지원 이메일**: 이메일 선택
     - **개발자 연락처 정보**: 이메일 입력
   - **"저장 후 계속"** 클릭
   - **범위** 페이지에서 **"저장 후 계속"** 클릭 (범위를 수동으로 추가할 필요 없음)
   - **테스트 사용자** 페이지에서 **"저장 후 계속"** 클릭 (나중에 테스트 사용자를 추가할 수 있음)
   - 검토 후 **"대시보드로 돌아가기"** 클릭

4. **"OAuth 클라이언트 ID 만들기"** 대화 상자에서:
   - **애플리케이션 유형**: **"데스크톱 앱"** 선택
   - **이름**: 이름 입력 (예: "WUG Desktop Client")
   - **"만들기"** 클릭

5. 인증 정보가 생성됩니다. 다음 단계에서 JSON 파일로 다운로드합니다.

### 4단계: 인증 정보 JSON 파일 다운로드

1. **"API 및 서비스"** > **"사용자 인증 정보"**에서 OAuth 2.0 클라이언트 ID 찾기
2. 클라이언트 ID 옆의 **다운로드 아이콘** (⬇️) 클릭
3. JSON 파일이 다운로드됩니다 (예: `client_secret_123456789-abc.apps.googleusercontent.com.json`)
4. **파일 이름을 `credentials.json`으로 변경**하고 홈 디렉토리에 배치 (예: `mv ~/Downloads/client_secret_123456789-abc.apps.googleusercontent.com.json ~/credentials.json`)

**중요 사항:**
- 파일 이름은 정확히 `credentials.json`이어야 하며 홈 디렉토리에 배치해야 합니다
- `credentials.json` 파일을 안전하게 보관하고 버전 관리에 커밋하지 마세요 (이미 `.gitignore`에 포함됨)

## 사용법

### 기본 사용법 (모든 새 파일 업로드)

```bash
# 바이너리를 다운로드하고 'wug'로 이름을 변경한 경우
./wug /path/to/directory

# 또는 PATH에 설치한 경우
wug /path/to/directory

# Windows
wug.exe C:\path\to\directory
```

이렇게 하면 파일이 Google Drive 루트에 업로드됩니다.

### 필터 옵션 사용 (특정 파일 확장자만 업로드)

```bash
./wug /path/to/directory --filter "*.safetensors"
```

또는 짧은 옵션 사용:

```bash
./wug /path/to/directory -f "*.txt"
```

### 특정 Google Drive 폴더에 업로드

Google Drive의 특정 폴더에 파일을 업로드하려면 `--path` 또는 `-p` 옵션과 함께 폴더 이름 또는 ID를 사용하세요:

```bash
# 폴더 이름 사용 (존재하지 않으면 자동 생성)
./wug /path/to/directory --path "MyUploads"
```

또는 짧은 옵션 사용:

```bash
./wug /path/to/directory -p "MyUploads"
```

**폴더 이름 사용:**
- 폴더가 존재하지 않으면 Google Drive 루트 디렉토리에 자동으로 생성됩니다
- 예시: `./wug /path/to/directory -p "Backups"` - "Backups" 폴더를 생성하거나 사용합니다

**폴더 ID 사용 (고급):**
- 알고 있는 경우 폴더 ID를 직접 사용할 수도 있습니다
- 예시: `./wug /path/to/directory -p "1a2b3c4d5e6f7g8h9i0j"`
- **Google Drive 폴더 ID 찾는 방법:**
  1. 웹 브라우저에서 Google Drive 열기
  2. 파일을 업로드할 폴더로 이동
  3. 폴더를 클릭하여 열기
  4. 브라우저 주소창의 URL 확인
  5. 폴더 ID는 URL의 `/folders/` 뒤에 있는 긴 문자열입니다
     - 예시 URL: `https://drive.google.com/drive/folders/1a2b3c4d5e6f7g8h9i0j`
     - 폴더 ID: `1a2b3c4d5e6f7g8h9i0j`

**참고**: 셸 glob 확장을 방지하기 위해 필터 패턴을 항상 따옴표로 감싸세요 (예: `"*.txt"`).

## 첫 실행 인증

프로그램을 처음 실행하면 인증 URL이 표시됩니다:

1. 표시된 URL을 브라우저에서 열기
2. Google 계정으로 로그인하고 권한 부여
3. "Google에서 이 앱을 확인하지 않았습니다"가 나타나면 **"계속"** 버튼 클릭
4. 권한을 부여한 후 `http://localhost`로 리다이렉션됩니다 - **이것은 정상적인 동작입니다**
5. 인증 코드는 브라우저 주소창 URL에 있습니다. URL은 다음과 같습니다:
   ```
   http://localhost/?state=state-token&code=4/0A0d-example_code_AbCdEf123456_GhIjKl789012_MnOpQr345678&scope=https://www.googleapis.com/auth/drive.file
   ```
6. **인증 코드만 복사** - `code=` 뒤부터 다음 `&` 기호 전까지의 부분입니다.
   - 위 예시에서는 다음을 복사합니다: `4/0A0d-example_code_AbCdEf123456_GhIjKl789012_MnOpQr345678`
   - **중요**: 코드 값만 복사하고, 전체 URL이나 `code=` 접두사는 포함하지 마세요
7. 프롬프트가 나타나면 터미널에 인증 코드 붙여넣기
8. 인증 토큰이 홈 디렉토리의 `token.json`에 저장됩니다 (이후 실행 시 자동으로 사용됨)

**참고**: 테스트 OAuth 앱을 사용하는 경우, OAuth 동의 화면 설정에서 Google 계정을 테스트 사용자로 추가해야 할 수 있습니다.

## 작동 방식

1. 프로그램이 지정된 디렉토리에서 새 파일을 감시합니다
2. 새 파일이 생성되면 자동으로 감지됩니다
3. 지능형 파일 쓰기 완료 감지 메커니즘을 사용하여 파일이 완전히 작성될 때까지 대기합니다
4. 필터가 지정된 경우 파일 확장자를 확인합니다
5. 파일이 Google Drive에 업로드됩니다

### 파일 쓰기 완료 감지

프로그램은 고정된 타임아웃 없이 파일 쓰기 완료를 정확하게 감지하기 위해 **디바운스 + rename 패턴**을 사용합니다:

#### 1. **Write 이벤트 디바운싱**
- 파일 시스템 감시자로부터 `Write` 이벤트를 모니터링합니다
- 각 파일의 마지막 쓰기 이벤트 타임스탬프를 추적합니다
- 쓰기 이벤트가 2초 동안 멈추면 (디바운스 지연) 파일이 준비된 것으로 간주됩니다
- 이는 점진적으로 또는 청크 단위로 작성되는 파일을 처리합니다

#### 2. **Rename 이벤트 감지**
- 많은 애플리케이션이 임시 파일에 먼저 쓰고 나중에 최종 파일명으로 이름을 변경합니다
- `Rename` 이벤트가 감지되면 파일이 즉시 준비된 것으로 간주됩니다
- 이는 원자적 쓰기 패턴을 사용하는 애플리케이션에 즉각적인 감지를 제공합니다

#### 3. **결합된 접근 방식**
- **Create 이벤트**: 파일 쓰기 완료를 모니터링하기 시작합니다
- **Write 이벤트**: 마지막 쓰기 타임스탬프를 업데이트합니다 (디바운스 메커니즘)
- **Rename 이벤트**: 파일을 즉시 준비된 것으로 표시합니다 (원자적 쓰기 패턴)
- 다음 중 하나가 발생하면 파일이 업로드 대기열에 추가됩니다:
  - 쓰기 이벤트가 2초 동안 멈춤 (디바운스), 또는
  - Rename 이벤트가 감지됨

#### 장점
- **고정 타임아웃 없음**: 모든 크기의 파일에 자동으로 적응합니다
- **빠른 감지**: Rename 이벤트가 원자적 쓰기에 즉각적인 감지를 제공합니다
- **신뢰성**: 디바운싱은 업로드 전에 파일이 완전히 작성되도록 보장합니다
- **효율성**: 작은 파일과 큰 파일 모두에서 잘 작동합니다 (예: safetensors 파일)

이 접근 방식은 특히 다음에 효과적입니다:
- 쓰는 데 시간이 걸리는 큰 파일 (모델 파일, 데이터셋 등)
- 원자적 쓰기를 사용하는 애플리케이션 (임시 파일 → rename)
- 점진적으로 또는 청크 단위로 작성되는 파일
- 쓰기 타이밍이 달라질 수 있는 네트워크 파일 시스템

## 문제 해결

### "credentials.json 파일을 찾을 수 없습니다"
- Google Cloud Console에서 인증 정보 JSON 파일을 다운로드했는지 확인하세요
- 다운로드한 파일 이름을 정확히 `credentials.json`으로 변경하세요
- 파일을 홈 디렉토리에 배치하세요

### "OAuth2 구성 실패"
- JSON 인증 정보가 유효한지 확인하세요
- 모든 필수 필드가 있는지 확인하세요 (client_id, client_secret, auth_uri, token_uri 등)

### "액세스 차단: 이 앱의 요청이 유효하지 않습니다"
- Google Cloud 프로젝트에서 Google Drive API를 활성화했는지 확인하세요
- OAuth 동의 화면이 올바르게 구성되었는지 확인하세요
- 테스트 앱을 사용하는 경우 Google 계정을 테스트 사용자로 추가했는지 확인하세요

### "redirect_uri_mismatch" 오류
- 인증 정보의 리다이렉트 URI가 애플리케이션이 예상하는 것과 일치해야 합니다
- 데스크톱 앱의 경우 일반적으로 `http://localhost`를 사용합니다
- Google Cloud Console에서 OAuth 클라이언트 구성을 확인하세요

## 참고 사항

- 프로그램 시작 시 이미 존재하는 파일은 업로드되지 않습니다 (새로 생성된 파일만 업로드됨)
- 프로그램은 고정된 타임아웃 대신 지능형 파일 쓰기 감지 (디바운스 + rename 패턴)를 사용합니다
- `credentials.json`과 `token.json` 파일은 홈 디렉토리에 저장되며 민감한 정보를 포함하므로 Git에 커밋하지 마세요 (이미 `.gitignore`에 포함됨)

## 라이선스

자세한 내용은 LICENSE 파일을 참조하세요.

